# Running goxsd8 on Claude Routines

The development loop is driven by scheduled Claude Code routines. Each
routine invokes exactly **one slash command** from `.claude/commands/` —
the same commands you can run locally on demand, so a scheduled run and an
interactive `/develop` behave identically.

## Schedule

| Routine | Command | When (UTC, local EST) | Purpose |
|---|---|---|---|
| backlog | `/backlog` | daily 12:00 (local 08:00) | reconcile issues, keep 5–10 `ready` |
| develop | `/develop` | 18:00, 00:00, 06:00 (local 14:00, 20:00, 02:00) | one issue → one commit |
| retro | `/retro` | weekly, Sun 13:00 (local Sun 09:00) | process self-improvement |
| ratchet | `/ratchet` | on demand | conformance maintenance |

Create them with the `/schedule` skill (or the Claude routines UI), one
routine per row, prompt = the slash command. **Routine cron is UTC** —
translate the local times above and mind DST drift.

## Environment requirements (cloud or local)

- Go ≥ 1.26; `golangci-lint` for the lint gate.
- `git submodule update --init testdata/xsdtests` (~215 MB, pinned W3C
  suite) — conformance runs skip without it.
- Non-interactive `git push` (credentials installed; a push that prompts
  hangs a headless session forever).
- **A GitHub channel for issue operations** — three options, in order of
  preference per context:
  1. *Cloud sessions*: the platform's built-in GitHub tools (read
     issues, list PRs, post comments) work with zero setup via the
     Claude GitHub app — the token never enters the container. Prefer
     these in routines.
  2. *GitHub MCP server* (`.mcp.json`): authenticates interactively via
     OAuth in local sessions; headless containers have no browser, so
     set a `GITHUB_PAT` environment variable in the routine's
     environment config — the committed config expands it into the
     Authorization header (`${GITHUB_PAT}`). Without the variable the
     server simply fails to connect; the other channels still work.
  3. *`gh` CLI*: authenticated locally; in cloud containers it needs
     installing (environment setup script) plus a `GH_TOKEN` env var.
- Cloud containers cannot delete or force-push remote refs (the git
  proxy rejects both). The workflow is structured so no session ever
  needs to: landing cleanup is GitHub's auto-delete on merge (repo
  setting "Automatically delete head branches" — keep it ON), and
  abandoned branches are retired in place.
- No human is watching: commands must never wait for input. Abort and log
  instead.

## Canonical commands (what the loop runs)

```sh
go build ./... && go test ./... && go vet ./...   # gate, part 1
golangci-lint run                                  # gate, part 2 (STYLE lint subset)
go test ./conformance -run TestConformance -count=1                 # conformance check
GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1 # ratchet (arbiter only)
go generate ./...                                  # regenerate spec md + catalogs + tables
go tool fetchspecs                                 # (re)download pristine spec HTML
```

## Ephemeral containers, overlap, and failure

- **Assume every routine run starts in a fresh container with a fresh
  clone.** Local git state — stashes, dirty trees, local-only branches,
  `.agent/` scratch — does not survive between runs. Anything not
  pushed does not exist (PRINCIPLES 28).
- **In-flight work is discovered from the branch namespace**, not from
  memory or comments: `git ls-remote --heads origin 'refs/heads/wip/*'`
  lists every attempt (`wip/issue-<N>`, one per issue); an attempt is
  resumable when its issue is open, not `needs-replan`, and its lease
  has expired. The develop loop resumes those before starting new
  issues. See the branch scheme in docs/WORKFLOW.md (normative).
- A failed, interrupted, or timed-out run is recoverable by design:
  durable state = the issue thread (grounding/verdict/RESUME comments) +
  the checkpointed WIP branch + main. A run that dies mid-session loses
  at most the work since its last checkpoint push; the next run's
  survey step finds the branch and continues.
- If two routines fire concurrently, the claim mechanism arbitrates: a
  `wip/issue-<N>` pushed within the 2h claim TTL is LIVE and off-limits
  to other sessions; simultaneous claims of the same issue are settled
  by git itself (the second push is rejected — that session picks a
  different issue). Checkpoint pushes refresh the lease, so a crashed
  session's branch expires on its own and the next run resumes it. Keep
  develop slots ≥ 6h apart so overlap stays rare anyway.
