# Development Workflow

goxsd8 is developed by Claude Code sessions — scheduled Claude routines
and on-demand local runs execute the **same slash commands**, so automated
and interactive behavior are identical by construction. A session is
short; the repo and its GitHub issues are the brain.

**Long-horizon memory lives in GitHub issues** (plans, groundings,
verdicts, hand-off notes — all as issue comments) **and docs/LOG/**
(history). Anything a future cold-start session needs must be on the issue
thread or in the log, never only in a transcript. Use the GitHub MCP
server for issue operations; the `gh` CLI is the local fallback.

## The cast

| Agent | File | Model | Role |
|---|---|---|---|
| **mason** | `.claude/agents/mason.md` | opus | Implements one issue at a time |
| **arbiter** | `.claude/agents/arbiter.md` | opus | Judges changes; owns the ratchet verdict |
| **oracle** | `.claude/agents/oracle.md` | sonnet | Spec exegesis; answers only from `docs/specs/md` |
| **warden** | `.claude/agents/warden.md` | opus | API/type-safety review; illegal states unrepresentable |
| **cartographer** | `.claude/agents/cartographer.md` | opus | Long-horizon planning; owns GitHub issues/milestones |
| **chronicler** | `.claude/agents/chronicler.md` | haiku | Session logs; meta-process retrospectives |
| **libuser** | `.claude/agents/libuser.md` | sonnet | Role-plays a library consumer; works only from godoc + README |
| **cliuser** | `.claude/agents/cliuser.md` | sonnet | Role-plays a CLI user; works only from README + `-help` |

Roles do not blur: mason never re-baselines the ratchet, arbiter never
implements fixes, oracle never writes code, libuser/cliuser never read
the source. The orchestrating session delegates and coordinates; it does
no specialist work itself and never skips the arbiter.

## The develop loop (`/develop`, the default scheduled trigger)

1. **Rescue** — a dirty tree at session start is stashed
   (`git stash push -u -m "rescue <timestamp>"`) and logged; never
   cleaned (PRINCIPLES 28).
2. **Pick** — list `ready` issues; take the highest-priority one whose
   dependencies are closed. No ready issue → run the cartographer
   instead and stop.
3. **Ground** — ask the **oracle** for the exact spec clauses and rule
   IDs in scope. Post the answer verbatim as a comment on the issue
   (`GROUNDING:` prefix); also save to `.agent/grounding-issue-<N>.md`
   as session scratch. The citation goes in the commit.
4. **Implement** — **mason** makes the smallest change that closes the
   issue, per docs/STYLE.md. New/changed public API → **warden** reviews
   before proceeding (post the verdict on the issue).
5. **Judge** — **arbiter** runs the gate
   (`go build ./... && go test ./... && go vet ./...` + the lint gate +
   the conformance run), reviews the diff against STYLE.md including the
   exported-surface diff (T5), and posts a verdict on the issue:
   - *accept* → arbiter runs the ratchet (`GOXSD_RATCHET=1`, upward only).
   - *reject* → one repair round by mason (edit the flagged lines, don't
     rewrite), then re-judge. Second reject → stash the work, comment
     findings, relabel `needs-replan`. **Two rejections is the hard cap**
     (PRINCIPLES 30).
6. **Record & commit** — **chronicler** appends to
   `docs/LOG/<year>-<month>.md` FIRST; then one commit carries the code
   and the log entry together; close or comment the issue; push. The tree
   is clean after every push — a session that leaves docs/LOG uncommitted
   has failed (PRINCIPLES 29).

Budget: one issue per session. Nothing works? A rescue stash + a good
issue comment is a successful session. Never wait for a human; abort
hanging commands and log the failure.

## Checkpoints & resume (context management)

The orchestrator's transcript is disposable; compaction may summarize it
at any moment. ALL durable session state lives on the issue thread and on
disk, written at step boundaries: the grounding comment, verdict comments,
rescue stashes, and commits. Compaction must never be able to eat anything
that can't be rebuilt from those.

Wrapping up early at a checkpoint (time budget hit, or second reject) is
a first-class outcome, not a failure. To hand off:

1. `git stash push -u -m "rescue #<N> <YYYYMMDD-HHMMSS>"`
2. Comment on the issue:

   ```
   RESUME: <last completed step, e.g. "implementation done, warden passed">
   Stash: rescue #<N> <timestamp>
   Next: <the exact next action, e.g. "arbiter verdict round 2 — prior
   findings were X, Y">
   Grounding: see the GROUNDING comment above (re-ask the oracle if absent)
   ```

3. Chronicler log entry, commit it, push.

To resume (next session, step 2 of the loop): find the stash named in the
newest RESUME comment via `git stash list`, then `git stash apply` (NOT
pop). If it applies cleanly, `git stash drop` that entry and continue
from "Next:". If it conflicts, undo the failed apply (`git checkout -- .`
is permitted here ONLY because the content is still safely held in the
stash), comment that the resume failed, and start the issue fresh — the
stash stays for human triage.

## Other triggers

- **`/ratchet`** — arbiter only: run conformance, report movement per
  lane, ratchet upward, investigate & file issues for any regression.
- **`/plan`** — cartographer: reconcile GitHub issues with reality (close
  stale, split oversized, order by dependency, keep 5–10 `ready`);
  consult **libuser**/**cliuser** when planning API- or CLI-facing
  milestones.
- **`/story`** — cartographer interviews libuser and cliuser (feeding
  them only the current README and `go doc` output) to produce user
  stories with acceptance criteria, filed as issues.
- **`/retro`** — chronicler: read the last ~2 weeks of LOG + git history +
  `needs-replan` issues; find recurring friction; apply the smallest
  durable fix to WORKFLOW/STYLE/agent prompts in a `meta: retro` commit.
  The ratchet-integrity rules (CLAUDE.md's one rule, arbiter.md's
  ratchet section) change only via a human-filed issue.

See docs/ROUTINES.md for the schedule. Every trigger is a slash command
you can also run locally on demand.

## GitHub conventions

- **Labels**: `ready` (unblocked, sized for one session), `blocked`,
  `needs-replan`, `epic`; areas
  `area/{model,xsderr,parser,value,builtin,xpath,validate,codegen,codec,regex,loader,conformance,cli,meta}`;
  kinds `kind/{feature,gap,bug,refactor,process,tooling,story}`.
- **Milestones** mirror docs/PLAN.md (M1, M2, …).
- Issue body must contain: goal, spec references (rule IDs), acceptance
  criteria (which conformance cases / tests prove it), and dependencies
  (`Depends on #N`). If an agent can't start it from the body alone, the
  body is incomplete.
- `// GAP(...)` comments and fail-open sites get tracking issues
  (`kind/gap`) so nothing fails open silently forever.

## Commit format

```
<area>: <what changed> (#<issue>)

Spec: <rule ids, or "n/a">
Ratchet: <lane movement, or "unchanged">
```

Small, focused, independently revertible. Ratchet expectation updates are
part of the same commit as the fix that earned them.

## The ratchet (the heart of the process)

- `conformance/testdata/expectations/*.txt`: one line per W3C test case,
  `<case-id> <expected-outcome>`, sorted, committed, one lane per file.
- `go test ./conformance -run TestConformance -count=1` fails if any case
  does worse than its expectation.
- The same run under `GOXSD_RATCHET=1` rewrites expectations for cases
  that now do better, refuses to write anything worse.
- Every expectation change must be explainable; "it flipped and I don't
  know why" blocks the commit and becomes an issue.

## Debugging playbook (for agents)

- Reproduce one failing conformance case in isolation before touching code
  (the harness supports single-case runs; see conformance's doc.go).
- Turn on scoped debug logs (`GOXSD_DEBUG=validate,xpath go test ...`) —
  messages carry rule IDs and locations by design.
- For bulk failure analysis, write an env-gated throwaway diagnostic test
  (`zz_diag_test.go`, gated on `DIAG=1`), harvest the pattern, delete it
  (PRINCIPLES 23).
- Grep the spec (`docs/specs/md/`), not your memory. Quote clauses in
  issues and commits.
- Friction with a manual process twice in a row? File a `kind/tooling`
  issue proposing a repo tool (PRINCIPLES 27).
