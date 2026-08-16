# Running goxsd8 on Claude Routines

The development loop is driven by scheduled Claude Code routines. Each
routine invokes exactly **one slash command** from `.claude/commands/` —
the same commands you can run locally on demand, so a scheduled run and an
interactive one behave identically.

## Schedule

| Routine | Command | When (UTC, local EST) | Purpose |
|---|---|---|---|
| backlog | `/backlog` | daily 12:00 (local 08:00) | reconcile issues, keep the ready queue dependency-ordered |
| develop | `/develop` | 18:00, 00:00, 06:00 (local 14:00, 20:00, 02:00) | one issue → one landing |
| retro | `/retro` | weekly, Sun 13:00 (local Sun 09:00) | process recraft + architecture audit |
| ratchet | `/ratchet` | on demand | conformance maintenance |

Create them with the `/schedule` skill (or the Claude routines UI), one
routine per row, prompt = the slash command. **Routine cron is UTC** —
translate the local times above and mind DST drift. Keep develop slots
≥ 6h apart so overlap stays rare.

## Environment requirements

- Go ≥ 1.26. Nothing else: the lint gate fetches its pinned linter with
  `go run`, which needs network access on a cold module cache.
- `git submodule update --init testdata/xsdtests` (~215 MB, pinned W3C
  suite) — conformance runs skip without it.
- Non-interactive `git push`: a push that prompts hangs a headless session
  forever.
- **A GitHub channel for issue operations.** Try them in this order, and
  **fall through on any error**: a channel that fails at call time means
  try the next one, never that the operation is impossible.
  1. *Cloud sessions*: the platform's built-in GitHub tools — zero setup
     via the Claude GitHub app, and the token never enters the container.
  2. *GitHub MCP server* (`.mcp.json`): OAuth interactively; headless
     containers have no browser, so set `GITHUB_PAT` in the routine's
     environment config and the committed config expands it into the
     Authorization header. Without it the server fails to connect.
  3. *`gh` CLI*: authenticated locally; cloud containers need it installed
     plus a `GH_TOKEN`.

  **In this environment `gh` is a standing 403 and the MCP server serves
  both reads and writes.** REST answers *"GitHub access is not enabled for
  this session. An org admin must connect the Claude GitHub App for this
  organization"*; the `gh` GraphQL path answers that only a pinned set of
  PR-review operations is served. Both name a configuration state rather
  than a transient one — recognize either text and fall through, rather
  than concluding the thread is unreadable or filing it again (#527).

  Falling through does not recover everything: the MCP channel strips
  angle-bracketed tokens from the issue bodies it reads, so nothing quoted
  through it is verbatim and anything bracketed is re-derived from the
  repo (#764).
- **The checkout is shallow.** History before the graft is absent, so
  `git merge-base` can come up empty — and
  `git rev-list --left-right --count A...B` does not fail when it does; it
  counts each side's whole visible history instead. An ahead/behind pair
  taken here is not a divergence measurement (#802).
- Cloud containers cannot delete or force-push remote refs — the git proxy
  rejects both. The workflow never needs to: landing cleanup is GitHub's
  auto-delete on merge, and abandoned branches are retired in place.
- No human is watching. Commands must never wait for input — abort and log
  instead.

The gate and the other canonical commands are defined once, in CLAUDE.md.
This file does not restate them.

## Failure and overlap

Assume every run starts in a fresh container with a fresh clone: anything
not pushed does not exist (PRINCIPLES 28). In-flight work is discovered
from the branch namespace, never from memory or comments — see the branch
scheme in docs/WORKFLOW.md, which is normative for leases, races and
retirement.

A failed, interrupted or timed-out run is recoverable by design: durable
state is the issue thread plus the checkpointed WIP branch plus `main`. A
run that dies mid-session loses at most the work since its last checkpoint
push, and the next run's survey finds the branch and continues.
