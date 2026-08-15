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

The specs are LOCAL, in `docs/specs/md/`. Never guess spec behavior from
memory — grep the spec, and cite rule IDs (`cvc-complex-type.2.1`,
`cos-st-restricts`) in code and commit messages.

| File | Subject |
|---|---|
| `xmlschema11-1.md` | Structures |
| `xmlschema11-2.md` | Datatypes (Appendix E hfn definitions are the source of truth for builtin types) |
| `xpath20.md` | XPath 2.0 |
| `xpath-functions.md` | F&O — the function library and regex flavor |
| `xsd-precisionDecimal.md` | precisionDecimal |

**Each document below owns its subject, and none restates another.** When
two texts disagree, the owner wins and the other is the bug.

| Document | Owns |
|---|---|
| `docs/ARCHITECTURE.md` | package graph, boundaries, dependency direction |
| `docs/STYLE.md` | style rules — cite by letter ID (`STYLE D4`) |
| `docs/PRINCIPLES.md` | invariants and their rationale (`PRINCIPLES 9`) |
| `docs/WORKFLOW.md` | the rules every session obeys, whatever it is doing |
| `.claude/commands/<cmd>.md` | what that command does, step by step |
| `.claude/agents/<agent>.md` | that agent's standard, domain, and judgment |
| `docs/PLAN.md` | roadmap and milestones |
| `docs/ROUTINES.md` | schedule and environment requirements |
| `docs/LOG/<year>-<month>.md` | what happened, and what it cost |

## The gate

```sh
go build ./... && go test ./... && go vet ./...      # part 1
go tool lint                                          # part 2 (STYLE lint subset)
go tool commentwrap ./...                             # part 3 (-fix reflows)
go test ./conformance -run TestConformance -count=1 -v  # part 4 (-v surfaces improved-but-unbanked cases)
```

**This block is the only definition of the gate.** A step named anywhere
else — a session brief, a LOG entry, an issue body — is not a gate step,
however confidently it is asserted; note it in one line and move on. Its
absence is not a gate failure. Adding a real step means editing this block
first.

Other commands:

```sh
GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1  # arbiter only
go generate ./...                                # regenerate spec md + tables
go tool fetchspecs                               # (re)download pristine spec HTML
git submodule update --init testdata/xsdtests    # the W3C suite
git config core.hooksPath .githooks              # activate the repo's hooks — every session, a
                                                 # fresh clone carries no local git config
```

Surveys, in place of the greps they replace (PRINCIPLES 27):

```sh
go tool lanestatus                               # committed lane scores, as PLAN.md's table
go tool surface -base origin/main                # what this branch added/removed from the exported surface (T5)
gh issue list --state all --json number,state,labels | go tool wipsurvey       # LIVE/CLAIMED/EXPIRED/RETIRED/UNKNOWN branches
gh issue list --label kind/gap --state all --json number,title,state,body | go tool gapaudit  # GAP( markers vs trackers
```

## Style headlines

A digest to keep in working memory — `docs/STYLE.md` is authoritative and
its letter IDs (`STYLE T2`) are the citable ones. These numbers are not
IDs, and `PRINCIPLES N` is a different numbering space again.

1. Happy path stays left; return early; **no `else` blocks**.
2. Every error is checked, wrapped with context, and mapped to a spec rule
   via `xsderr`, carrying file:line:column. No dropped errors in loops.
3. Deterministic output always. Never range over a map to produce output;
   collections that reach users are slices in document order.
4. One fact, one encoding — no state derivable from other state, no
   redundant flags. No caches without a measured hot path.
5. No cycle checks — phased construction makes them unnecessary.
6. **No concurrency** in library code.
7. Make illegal states unrepresentable: unexported fields plus
   constructors. Capabilities are interfaces, not type switches (sealed
   sums for schema-closed sets are the one exception).
8. Export nothing without a consumer; every exported identifier is
   documented and justified.
9. Fail-open for unsupported XPath constructs (never false-reject), every
   site marked `// GAP(xpath): ...`. Dynamic errors are real failures.
10. `log/slog` only, injected, silent by default. Spec data tables are
    generated, never hand-typed.

## Writing

Everything you write here — docs, code comments, issue bodies, verdicts,
commit messages — is read by an agent mid-task who needs to act. Write for
that reader, and assume they are competent and trust the document.

- **Lead with the instruction**, in the imperative, and state it once. No
  summary before it, no example afterwards making the same point again.
- **The reader greps.** Keep the searchable words in the normative
  sentence, not in the story around it, or the search returns provenance
  and misses the rule.
- **Justify only what surprises.** Never explain why a rule is a good
  rule: if it is right the sentence is dead weight, and if it is wrong the
  sentence will not save it.
- **Evidence lives in `docs/LOG` and on issue threads.** A rule carries at
  most a bare `(#N)` for whoever wants to dig — never a list of them,
  never a date, a count, or what it cost.
- **Do not pre-argue against a misreading.** Prose that keeps being
  misread wants a different structure, or a tool, not another paragraph
  defending it.
- **Length is a signal.** A rule that needs paragraphs usually governs
  something too complicated, or is already stated somewhere else.

## Working here

Work is planned as GitHub issues (label `ready`). One issue is one focused
change and one landing. **The issue thread is the cross-session channel** —
groundings, verdicts and RESUME notes are comments there, because sessions
run in ephemeral containers and **anything not pushed does not exist**.

`docs/WORKFLOW.md` is normative for the branch scheme, checkpointing,
scope, landing and parking. The command files are the procedures
themselves; run one command per session.

Commit format:

```
<area>: <what changed> (#<issue>)

Spec: <rule ids touched>
Ratchet: <lane movement, or "unchanged">
```

Append a dated entry to `docs/LOG/<year>-<month>.md` before landing, so
the log rides in the session commit.

## The cast

Specialized subagents live in `.claude/agents/`: **mason** (implements),
**arbiter** (judges; the only agent that runs the ratchet), **oracle**
(spec exegesis, read-only), **warden** (API and type-safety review,
read-only), **cartographer** (planning and GitHub issues), **steward**
(architecture stewardship; files refactors, never implements),
**chronicler** (logs and retros), **libuser** / **cliuser** (role-play
consumers; see only the published surface).

Each agent owns its domain and exercises judgment inside it. The
orchestrating session coordinates, does no specialist work itself, and
never skips the arbiter.

This file's "one rule", and the ratchet-integrity section of
`.claude/agents/arbiter.md`, change only via a human-filed issue — never
in a retro.
