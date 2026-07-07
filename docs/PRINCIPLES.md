# Engineering Principles

Numbered so reviews and issues can cite them ("PRINCIPLES 7"). These are
paid-for invariants: each one exists because the alternative demonstrably
hurts in an XSD processor. Re-litigating one without new evidence is a
process failure worth a chronicler entry. The enforceable subset is
codified as rules in docs/STYLE.md.

## Architecture

1. **The core model is a pure leaf.** The component model (`xsd`) and the
   error currency (`xsderr`) import nothing from this module. Every other
   package builds above them. An upward import from a leaf is an
   architecture bug, full stop.

2. **`Value` is open.** The value abstraction is `any` plus small
   capability interfaces, never a sealed interface. A sealed value type
   blocks user-supplied backends and custom value spaces — the exact thing
   the backend seam exists for.

3. **Capabilities over type switches.** The facet engine and comparison
   logic discover what a value can do (`Ordered`, `Lengthed`,
   `DigitCounted`, `Scaled`, `Identical`, `TimezoneAware`, `Canonical`)
   by interface narrowing. A type switch over concrete value types outside
   the defining package silently excludes every type it doesn't know.

4. **Minimal capability views at boundaries.** Consumers get the narrowest
   interface that serves them (a schema view exposing only
   `ElementByName`), not the whole object. Wide interfaces calcify.

5. **One fact, one encoding.** Never store a flag that re-states what
   other fields already imply. Canonical example: no `Primitive bool` —
   a type that defines its own fundamental facets *is* a primitive; expose
   `IsPrimitive()` as a derived method if callers want the question
   answered. Duplicate encodings drift the moment one write site forgets
   one of them; the hand-copying at clone/build sites IS the bug class.

6. **No derivable state, no speculative caches.** Effective facets,
   transitive membership, mixedness — compute on demand from the canonical
   source. A memoized cache is a liability until a profile proves a hot
   path.

7. **No concurrency.** Parsing and validation are pure transforms; they
   stay single-threaded. Concurrency buys nondeterminism, lock invariants,
   and race classes — for a workload that is not parallel-bound. If a
   measured need ever appears, it goes behind one seam.

8. **The marker-interface infoset scales across sources.** Instance
   validation consumes an abstract element/attribute/node view, so XML,
   JSON, and BER plug in as adapters without the engine importing any of
   their decoders.

9. **Phased construction beats cycle checks.** Parse raw, resolve names,
   finalize in dependency order — then no traversal ever needs a `seen`
   set. Where the spec permits reference cycles it also names the rule
   that forbids the harmful ones; detect those once, at finalize, with
   that rule ID.

## Spec traps (each of these produces wrong verdicts if ignored)

10. **There are two regex flavors, one grammar family.** The F&O
    (XPath/XQuery Functions & Operators) regex grammar is a superset of
    the XSD pattern-facet grammar — so one engine serves both — but their
    semantics differ: XSD patterns are implicitly anchored, treat `^`/`$`
    as literal characters, have non-capturing groups and no flags; F&O is
    unanchored (a match anywhere succeeds), has real anchors, capturing
    groups (`$N` in `fn:replace`), and flags (`i s m x q`). Assertion
    `fn:matches`/`fn:replace`/`fn:tokenize` bind to the F&O flavor, never
    the pattern-facet flavor. Validation of a pattern is flavor-scoped:
    a construct one flavor cannot express is an error in that flavor, not
    a silent accept.

11. **Union validation uses direct members.** A union validates against
    its DirectMembers in order — not a flattened member list, because
    intervening restrictions carry facets — and a pattern facet on the
    union is matched against the value as normalized by the *validating
    member's* whiteSpace.

12. **Assertions live at every variety level.** Atomic, list (per item
    and per list), union (per member and on the union) — miss a level and
    a class of schemas validates wrongly.

13. **Empty content is stricter than element-only.** A complex type whose
    particle can never match an element admits NO character content, not
    even whitespace (element-only allows whitespace).

14. **Content matching is greedy and deterministic.** Unique Particle
    Attribution makes the content model unambiguous, so the matcher never
    backtracks — and explicit content always beats an open-content
    wildcard at the current state.

15. **Identity-constraint matching is namespace-stateful.** Selector and
    field XPaths resolve prefixes — and the default element namespace from
    `xpathDefaultNamespace` — against in-scope bindings; the default
    namespace applies to element steps, never attribute steps.

16. **`xs:override` needs explicit target tracking.** Components declared
    inside an override belong to the *overridden* document: its
    schema-level defaults apply, and suppression of replaced components
    must not leak back into the overriding document under mutual/circular
    overrides.

17. **XPath variables are typed atoms.** `$value` binds `{Lexical, Kind}`,
    not a bare string; comparisons and casts depend on the kind.

18. **precisionDecimal values keep their scale.** The value is a
    (coefficient, scale, sign) identity — `3`, `3.0`, `3.00` are distinct
    values that compare numerically equal; totalDigits counts trailing
    zeros. NaN is incomparable in the order but identical to itself for
    enumeration — identity and order are separate capabilities.

19. **QName and NOTATION need context at parse time.** The lexical→value
    mapping of a QName requires in-scope namespace bindings, so the
    mapping signature carries a context; a mapping that can't resolve
    returns a sentinel the engine detects, never a fabricated value.

## Strategy

20. **Fail-open partial XPath is correct — only with tracking.** An
    unsupported construct must never cause a false rejection; every
    fail-open site carries a greppable `// GAP(…)` marker and the
    conformance ratchet keeps the gap set shrinking. Fail-open without
    tracking is silent wrongness. And the error direction matters: a
    *dynamic* error (type mismatch, bad pattern) makes an assertion
    definitively unsatisfied — folding it into fail-open would flip a
    false-accept into a false-reject or vice versa.

21. **Ratchet expectations live in version control.** One committed line
    per conformance case; regressions are loud diffs; improvements are
    harvestable; `git blame` bisects verdict changes.

22. **Small fix → re-baseline → commit.** One issue, one focused change,
    one commit that carries its own expectation movement. Batched changes
    make ratchet movement unattributable.

23. **Throwaway diagnostics are first-class.** An env-gated `zz_diag`
    test that dumps every wrong verdict is often the fastest route to the
    next fix. Write it, harvest it, delete it before handoff.

24. **Fuzzing finds panics, not logic bugs.** Fuzz the parsers for
    robustness; rely on the conformance suites for correctness.

25. **Some suite cases exercise spec bugs.** When a test contradicts the
    spec text, record it as an expected divergence with a comment citing
    both — never contort the implementation to chase a broken case.

26. **Spec data tables are generated, never hand-typed.** Builtin type
    properties, facet applicability, rule catalogs: a deterministic
    generator reads the local spec and emits the table, committed together
    with byte-identical regeneration. Hand-transcription of 49 datatypes
    is where typos become conformance bugs.

27. **Build repo tools for repetitive work.** When a task is repetitive,
    deterministic, or error-prone by hand — extraction, cataloging,
    regeneration, diagnostics — write a small Go tool under `tools/`,
    register it in go.mod's `tool` block, and wire it to `go generate`
    where it produces artifacts. Tools are first-class deliverables with
    the same style bar as the library.

## Process

28. **Anything not pushed does not exist — and the branch NAME is the
    index.** Sessions run in ephemeral containers: the next run may be a
    fresh clone, so stashes, dirty trees, and local-only branches are
    already lost the moment the session ends. All work lives on pushed
    branches under a fixed scheme (`wip/issue-<N>` in-flight and
    resumable, one per issue, the name itself the claim;
    `parked/…-<ts>` abandoned, triage-only) so any cold-start agent
    reconstructs the in-flight state from one
    `git ls-remote --heads origin 'refs/heads/wip/*'` — discovery must
    never depend on reading comments or transcripts. The claim has a
    lease: a `wip/` tip pushed within the 2h TTL is live and
    off-limits; older is resumable — and simultaneous pushes are
    arbitrated by git's atomic ref updates (rejected push = lost race;
    force-pushing `wip/*`/`parked/*` is forbidden). Checkpoint
    (commit + push) at every step boundary — it is also the lease
    heartbeat. And never destroy
    uncommitted work: `git clean`, `git restore .`,
    `git checkout -- <file>`, and stashing of any kind are forbidden —
    a dirty local tree is pushed to `parked/untriaged-<ts>` and logged.

29. **The session log rides in the session commit.** A log entry written
    after the commit is left uncommitted and dies with the next session's
    cleanup. Chronicler writes first; then one commit carries code + log.

30. **Two rejections is the convergence horizon.** If the arbiter rejects
    the same change twice, a third attempt will not converge: park the
    WIP branch (`parked/issue-<N>-<ts>`), comment findings on the issue,
    relabel `needs-replan`, stop. A fresh attempt after re-planning
    starts from main, never from the parked branch.

31. **Documentation is the tested product surface.** The README and the
    package godoc are what the user personas (libuser, cliuser) work from
    — exclusively. A gap they hit is a documentation bug by definition,
    filed like any other bug.

32. **Ratchet numbers live in expectations files only.** Never copy
    baseline counts into prose docs; duplicated numbers drift (see 5).
    Prose refers to lanes by name; the files are the truth.
