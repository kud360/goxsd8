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

- Go ≥ 1.26, and `golangci-lint` for the lint gate.
- `git submodule update --init testdata/xsdtests` (~215 MB, pinned W3C
  suite) — conformance runs skip without it.
- Non-interactive `git push`: a push that prompts hangs a headless session
  forever.
- **A GitHub channel for issue operations**, in order of preference:
  1. *Cloud sessions*: the platform's built-in GitHub tools — zero setup
     via the Claude GitHub app, and the token never enters the container.
     Prefer these in routines.
  2. *GitHub MCP server* (`.mcp.json`): OAuth interactively; headless
     containers have no browser, so set `GITHUB_PAT` in the routine's
     environment config and the committed config expands it into the
     Authorization header. Without it the server simply fails to connect
     and the other channels still work.
  3. *`gh` CLI*: authenticated locally; cloud containers need it installed
     plus a `GH_TOKEN`.
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
