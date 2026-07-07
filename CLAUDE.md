# goxsd8 — Agent Instructions

You are working on **goxsd8**, a conformance-grade XSD 1.1 processor in Go
(module `github.com/kud360/goxsd8`). This repo is developed primarily by AI
agents. Follow this file exactly; it wins over your own preferences.

## The one rule that outranks everything

**Never regress the ratchet.** Conformance expectations live in
`conformance/testdata/expectations/`. Scores only move up. If your change
makes a previously passing case fail, either fix it or revert your change.
Never edit an expectations file downward to make CI green.

## Ground truth

- The specs are LOCAL, in `docs/specs/md/`. Do not guess spec behavior from
  memory — grep the spec. Cite rule IDs (e.g. `cvc-complex-type.2.1`,
  `cos-st-restricts`) in code and commit messages when implementing them.
  - `xmlschema11-1.md` — Structures
  - `xmlschema11-2.md` — Datatypes (Appendix E hfn function definitions
    are the source of truth for builtin types)
  - `xpath20.md` — XPath 2.0
  - `xpath-functions.md` — F&O (the XPath 2.0 function library and regex flavor)
  - `xsd-precisionDecimal.md` — precisionDecimal
- Architecture: `docs/ARCHITECTURE.md`. Style: `docs/STYLE.md`
  (non-negotiable). Invariants & rationale: `docs/PRINCIPLES.md`.
  Workflow: `docs/WORKFLOW.md`. Roadmap: `docs/PLAN.md`.
  Schedules: `docs/ROUTINES.md`.

## Commands (no Make — Go subcommands only)

```sh
go build ./... && go test ./... && go vet ./...    # gate, part 1
golangci-lint run                                   # gate, part 2 (STYLE lint subset)
go test ./conformance -run TestConformance -count=1                  # conformance check
GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1  # ratchet — ARBITER ONLY
go generate ./...                                   # regenerate spec md + generated tables
go tool fetchspecs                                  # (re)download pristine spec HTML
```

The full gate (build, test, vet, lint, conformance) must pass before any
commit. W3C suite: `git submodule update --init testdata/xsdtests`.

## Style headlines (full rules in docs/STYLE.md — cite IDs in reviews)

1. Happy path stays left; return early; **no `else` blocks**.
2. Every error is checked, wrapped with context, and mapped to a spec
   validation rule via `xsderr`. Errors carry file:line:column.
   No dropped errors inside loops — collect or return.
3. Deterministic output always. Never range over a map to produce output.
   Collections that reach users are slices in document order.
4. One fact, one encoding: no state derivable from other state, no
   redundant flags (no `Primitive bool` beside fundamental facets — use a
   derived method). No caches without a measured hot path.
5. No cycle checks — phased construction makes them unnecessary.
6. **No concurrency** in library code.
7. Make illegal states unrepresentable: unexported fields + constructors.
   Capabilities are interfaces, not type switches (sealed sums for
   schema-closed sets are the one exception).
8. Export nothing without a consumer; every exported identifier is
   documented and justified — reviews inspect the exported-surface diff.
9. Fail-open for unsupported XPath constructs (never false-reject), every
   fail-open site marked `// GAP(xpath): ...`. Dynamic errors are real
   failures, not fail-open.
10. `log/slog` only, injected, silent by default. Spec data tables are
    generated, never hand-typed.

## Workflow (full loop in docs/WORKFLOW.md)

Work is planned as GitHub issues (label `ready`). One issue = one focused
change = one commit. **GitHub issues are the cross-session channel**: post
oracle groundings, arbiter verdicts, and RESUME hand-off notes as issue
comments so any later session can reconstruct context from the thread.
Use whichever GitHub channel this session has — cloud built-in GitHub
tools, the GitHub MCP server (needs `GITHUB_PAT` when headless), or
`gh` CLI (see docs/ROUTINES.md). Commit format:

```
<area>: <what changed> (#<issue>)

Spec: <rule ids touched>
Ratchet: <lane movement, or "unchanged">
```

Append a dated entry to `docs/LOG/<year>-<month>.md` BEFORE landing so
the log rides in the session commit. Sessions run in ephemeral
containers: **anything not pushed does not exist**. All work happens on
pushed WIP branches under the fixed scheme (docs/WORKFLOW.md is
normative): `wip/issue-<N>` is THE branch for issue #N (stable name =
the claim, tip time = the lease: pushed within 2h → live, off-limits;
older → resumable; rejected push = lost race, never force-push;
discover in-flight work with
`git ls-remote --heads origin 'refs/heads/wip/*'`); checkpoint
(commit + push) at every step boundary; land by opening a PR
(`Closes #<N>` in the body) and squash-merging it via the GitHub Merge
API as one commit (GitHub auto-deletes the branch); abandoned attempts
are retired in place under `needs-replan`, never resumed. Never stash,
never destroy a dirty tree. Two arbiter rejections is the hard cap; then park, comment,
relabel `needs-replan`, stop.

## Personas

Specialized subagents are defined in `.claude/agents/`:
**mason** (implements), **arbiter** (judges & runs the ratchet — the only
agent allowed to), **oracle** (spec exegesis, read-only), **warden**
(API/type-safety review, read-only), **cartographer** (long-horizon
planning, GitHub issues), **chronicler** (logs & retros), **libuser**
(role-plays a library consumer; sees only godoc + README), **cliuser**
(role-plays a CLI user; sees only README + `-help`). Delegate per
docs/WORKFLOW.md; do not blur their roles. The orchestrating session
coordinates, does no specialist work, and never skips the arbiter.

This file's "one rule" and the arbiter's ratchet-integrity section change
only via a human-filed issue — never in a retro.
