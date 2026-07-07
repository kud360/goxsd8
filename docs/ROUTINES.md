# Running goxsd8 on Claude Routines

The development loop is driven by scheduled Claude Code routines. Each
routine invokes exactly **one slash command** from `.claude/commands/` —
the same commands you can run locally on demand, so a scheduled run and an
interactive `/develop` behave identically.

## Schedule

| Routine | Command | When (local) | Purpose |
|---|---|---|---|
| plan | `/plan` | daily 08:00 | reconcile issues, keep 5–10 `ready` |
| develop | `/develop` | 14:00, 20:00, 02:00 | one issue → one commit |
| retro | `/retro` | weekly, Sun 09:00 | process self-improvement |
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
- GitHub MCP server connected (`.mcp.json`) for issue/label/milestone
  operations; `gh` CLI works as local fallback.
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
  lists every resumable attempt (`wip/issue-<N>`, one per issue); the
  develop loop resumes those before starting new issues. See the branch
  scheme in docs/WORKFLOW.md (normative).
- A failed, interrupted, or timed-out run is recoverable by design:
  durable state = the issue thread (grounding/verdict/RESUME comments) +
  the checkpointed WIP branch + main. A run that dies mid-session loses
  at most the work since its last checkpoint push; the next run's
  survey step finds the branch and continues.
- If two routines fire concurrently, the WIP branch is the claim: the
  second run sees `wip/issue-<N>` freshly pushed and picks a different
  `ready` issue (or stops). Keep develop slots ≥ 6h apart to make even
  that rare.
