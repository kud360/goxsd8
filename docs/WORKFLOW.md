# Development Workflow

goxsd8 is developed by Claude Code sessions — scheduled Claude routines
and on-demand local runs execute the **same slash commands**, so automated
and interactive behavior are identical by construction. A session is
short; the repo and its GitHub issues are the brain.

**Long-horizon memory lives in GitHub issues** (plans, groundings,
verdicts, hand-off notes — all as issue comments) **and docs/LOG/**
(history). Anything a future cold-start session needs must be on the issue
thread or in the log, never only in a transcript. For issue operations
use whichever GitHub channel the session has (docs/ROUTINES.md ranks
them): the cloud platform's built-in GitHub tools, the GitHub MCP server
(needs `GITHUB_PAT` when headless), or the `gh` CLI.

**The container is ephemeral.** A scheduled routine may start from a
fresh clone every run: local git state — stashes, dirty trees, local-only
branches, `.agent/` scratch — does NOT survive between sessions. The rule
that follows: **anything not pushed does not exist** (PRINCIPLES 28).
All work therefore happens on pushed WIP branches under a fixed naming
scheme; the scheme itself is how a cold-start session discovers in-flight
work.

## The branch scheme (the WIP discovery index)

The remote branch namespace is a machine-readable index. Exactly three
kinds of branches exist; nothing else is ever pushed:

| Branch | Meaning | Lifecycle |
|---|---|---|
| `main` | always green; receives only squash-landed, arbiter-accepted develop work plus the maintenance triggers' own commits (`meta:`/`conformance:` from /backlog, /retro, /ratchet) | permanent |
| `wip/issue-<N>` | THE work branch for issue #N — at most one, its name is stable | created when work starts; auto-deleted by GitHub when its PR squash-merges; retired in place if the attempt is abandoned |
| `parked/untriaged-<YYYYMMDD-HHMMSS>` | unattributable work found in a dirty local tree | kept for human triage |

Sessions only ever CREATE refs; the one deletion in the system is
GitHub's auto-delete on merge. Any session can reconstruct the entire
in-flight state with one command, no issue archaeology required:

```sh
git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'
```

Invariants the scheme encodes:

- **The name is the claim; the tip time is the lease.** `wip/issue-<N>`
  existing means issue #N has an in-flight attempt, and its stable name
  makes a second branch for the same issue impossible to create by
  accident. But existence alone cannot distinguish "being worked right
  now" from "abandoned by a dead session" — that is the tip's committer
  timestamp (`git log -1 --format=%cI origin/wip/issue-<N>`):
  - tip newer than the **claim TTL (2 hours)** → LIVE: another session
    presumably holds it. Do not resume it, and do not start issue #N.
  - tip older than the TTL → EXPIRED: the owner is gone (a healthy
    session checkpoints far more often than that) → resumable.
  Checkpoint pushes are therefore also the lease heartbeat: a step
  expected to run long pushes intermediate commits rather than letting
  its lease lapse mid-work.
- **Races are settled by git's atomic ref updates — never by force.**
  Two sessions claiming or checkpointing the same branch: the second
  push is rejected (non-fast-forward). A rejected push to `wip/*` means
  you lost the race — fetch, abandon your local copy of that attempt,
  and pick a different issue. Force-pushing any `wip/*` or `parked/*`
  ref is forbidden; it is the one way sessions could actually stomp
  each other.
- **A `wip/` branch is work only while its issue is open and not
  `needs-replan`.** Otherwise it is retired: landed branches vanish on
  merge, abandoned ones stay put as re-planning evidence — never
  resumed, never deleted by a session. A retired name is never
  contended, because an issue is never re-attempted under its own
  number: re-planning supersedes it with a new issue, and the fresh
  attempt starts as `wip/issue-<M>` from `origin/main`.
- **Checkpoint = commit + push.** Work is committed on the WIP branch at
  every step boundary (grounding done, implementation done, each verdict)
  with message `wip #<N>: <step>`, and pushed immediately. A session that
  dies loses at most the work since its last checkpoint — and its lease
  expires on its own, so the next session recovers the branch without
  human help.
- **Landing is atomic.** Accept → open a PR for the WIP branch and
  squash-merge it via the GitHub Merge API: the squash is the session's
  ONE commit (code + log entry), `Closes #<N>` in the PR body closes
  the issue, and GitHub auto-deletes the head branch (repo setting
  "Automatically delete head branches" — keep it ON). A `wip/issue-<N>`
  whose issue is closed is retired — survey skips it; the cartographer
  verifies its content is in main (`git log`/diff) and supersedes the
  issue if it isn't.

## The cast

| Agent | File | Model | Role |
|---|---|---|---|
| **mason** | `.claude/agents/mason.md` | opus | Implements one issue at a time |
| **arbiter** | `.claude/agents/arbiter.md` | opus | Judges changes; owns the ratchet verdict |
| **oracle** | `.claude/agents/oracle.md` | sonnet | Spec exegesis; answers only from `docs/specs/md` |
| **warden** | `.claude/agents/warden.md` | opus | API/type-safety review; illegal states unrepresentable |
| **cartographer** | `.claude/agents/cartographer.md` | opus | Long-horizon planning; owns GitHub issues/milestones |
| **steward** | `.claude/agents/steward.md` | opus | Long-horizon architecture steward; audit (Part 2 of /retro); pre/post-1.0 mobility policy |
| **chronicler** | `.claude/agents/chronicler.md` | opus | Session logs; meta-process retrospectives |
| **libuser** | `.claude/agents/libuser.md` | sonnet | Role-plays a library consumer; works only from godoc + README |
| **cliuser** | `.claude/agents/cliuser.md` | sonnet | Role-plays a CLI user; works only from README + `-help` |

Roles do not blur: mason never re-baselines the ratchet, arbiter never
implements fixes, oracle never writes code, libuser/cliuser never read
the source, steward never touches Go code (it files refactor issues;
warden judges individual diffs, steward judges the whole). The
orchestrating session delegates and coordinates; it does no specialist
work itself and never skips the arbiter.

## The develop loop (`/develop`, the default scheduled trigger)

1. **Survey** — `git fetch origin` and list the WIP index:
   `git ls-remote --heads origin 'refs/heads/wip/*'`. If the local tree
   is somehow dirty (persistent local checkout only; a routine container
   starts clean), push it to `parked/untriaged-<ts>` first and log it —
   never clean it (PRINCIPLES 28).
2. **Pick** — partition the `wip/*` branches by lease
   (tip timestamp vs the 2h claim TTL — see the branch scheme):
   - **LIVE** branches (and their issues) are off-limits this session.
   - **Resuming beats starting**: if any EXPIRED `wip/issue-<N>` exists
     whose issue is open and not `needs-replan`, take the oldest —
     switch to it, merge `origin/main` in if main has moved (never
     rebase — a rewritten branch cannot be pushed without force), read
     the issue's newest `RESUME:` comment, and continue from its
     "Next:". (A merge with non-trivial conflicts → park the branch,
     comment, pick again.)
   - Otherwise take the highest-priority `ready` issue with closed
     dependencies and no live branch, and claim it:

     ```sh
     git switch -c wip/issue-<N> origin/main
     git push -u origin HEAD     # the push IS the claim
     ```

     Push rejected → another session claimed it between your survey and
     now; fetch and pick again.
   - Nothing to resume, nothing ready → run the cartographer instead
     and stop.
3. **Ground** — ask the **oracle** for the exact spec clauses and rule
   IDs in scope. Post the answer verbatim as a comment on the issue
   (`GROUNDING:` prefix); also save to `.agent/grounding-issue-<N>.md`
   as session scratch. The citation goes in the commit. **Checkpoint.**
4. **Implement** — if the issue's `## Surface` section is non-"none",
   **warden** pre-flights the planned surface (sketch + intended type
   shapes) before any code is written; post it on the issue. Then
   **mason** makes the smallest change that closes the issue, per
   docs/STYLE.md, committing on the WIP branch. New/changed public API
   → **warden** reviews the diff before proceeding (post the verdict
   on the issue). **Checkpoint.**
5. **Judge** — **arbiter** runs the gate
   (`go build ./... && go test ./... && go vet ./...` + the lint gate +
   the conformance run), reviews the branch diff against main per
   STYLE.md including the exported-surface diff (T5), and posts a
   verdict on the issue. **Checkpoint after each verdict.**
   - *accept* → arbiter runs the ratchet (`GOXSD_RATCHET=1`, upward only).
   - *reject* → one repair round by mason (edit the flagged lines, don't
     rewrite), then re-judge. Second reject → **park** (see below),
     comment findings, relabel `needs-replan`. **Two rejections is the
     hard cap** (PRINCIPLES 30).
6. **Land** — **chronicler** appends to `docs/LOG/<year>-<month>.md` on
   the WIP branch FIRST (PRINCIPLES 29) and checkpoints; then land
   through GitHub, never a local merge:

   1. Open a PR from `wip/issue-<N>` to `main`; the body carries
      `Closes #<N>` plus a pointer to the arbiter's accept verdict.
   2. Squash-merge it via the Merge API (MCP `merge_pull_request` with
      `merge_method: "squash"`, or the platform's built-in PR tools),
      supplying the CLAUDE.md commit format as the squash title
      (`<area>: <what changed> (#<N>)`) and body (`Spec:`/`Ratchet:`
      lines).
   3. GitHub finishes server-side: main gets the ONE session commit,
      `Closes #<N>` closes the issue, and the head branch is
      auto-deleted.

   Nothing else is ever committed directly to main.

7. **Post-land pass** — the **cartographer**, twofold: (a) **unblock**
   — scan `blocked` issues whose `Depends on:` names the just-closed
   issue; any whose dependencies are now ALL closed is relabeled
   `ready`, with a one-line comment naming the landing that unblocked
   it; (b) **harvest follow-ups from this landing** — the session log
   entry's "Next:"/deferred items and the issue thread's advisory
   verdict notes are each filed as an issue or explicitly dismissed in
   a comment, while the context is fresh. The dependency graph and the
   follow-up ledger react to landings immediately instead of waiting
   for the next /backlog.

Budget: one issue per session. Nothing works? A checkpointed WIP branch
+ a good RESUME comment is a successful session. Never wait for a human;
abort hanging commands and log the failure.

## Checkpoints, hand-off, and parking

**Checkpoint** (at every step boundary, and before ending any session):

```sh
git add -A && git commit -m "wip #<N>: <step completed>"
git push origin wip/issue-<N>
```

plus a `RESUME:` comment on the issue whenever the next action isn't
obvious from the branch alone:

```
RESUME: <last completed step, e.g. "implementation done, warden passed">
Next: <the exact next action, e.g. "arbiter verdict round 2 — prior
findings were X, Y">
Grounding: see the GROUNDING comment above (re-ask the oracle if absent)
```

The branch carries the CONTENT; the RESUME comment carries the INTENT.
Discovery never depends on the comment — `wip/issue-<N>` is found by
listing the namespace — but a good "Next:" saves the resuming session
from re-deriving where things stood.

The orchestrator's transcript is disposable (compaction may summarize it
at any moment) and so is the container. ALL durable state lives on
GitHub: the issue thread, the pushed WIP branch, and main. Neither
compaction nor a recycled container may be able to eat anything that
can't be rebuilt from those. Wrapping up early at a checkpoint (time
budget hit, second reject) is a first-class outcome, not a failure.

**Park** (second reject, or a resume whose merge won't resolve):
checkpoint the branch one final time, label the issue `needs-replan`,
and comment the findings that killed the attempt. Nothing is renamed or
deleted — the `needs-replan` label alone retires the branch in place,
where it stays as re-planning evidence, not resumable work. After
re-planning, the cartographer closes the issue as superseded and files
a replacement; the fresh attempt starts as `wip/issue-<M>` under the
new number, from `origin/main`.

## Other triggers

- **`/ratchet`** — arbiter only: run conformance, report movement per
  lane, ratchet upward, investigate & file issues for any regression.
- **`/backlog`** — cartographer: reconcile GitHub issues with reality (close
  stale, split oversized, order by dependency, keep 8–10 `ready`);
  consult **libuser**/**cliuser** when planning API- or CLI-facing
  milestones. Also **reconcile the branch namespace**: classify every
  `wip/*` branch by its issue's state (live / resumable / retired); a
  `wip/` branch stale for several days with no RESUME comment gets its
  issue flagged `needs-replan`; a closed issue's leftover branch is
  verified landed (superseded if it isn't); retired branches and
  `parked/untriaged-*` are listed in the plan summary for human triage
  — never deleted by an agent.
- **`/story`** — cartographer interviews libuser and cliuser (feeding
  them only the current README and `go doc` output) to produce user
  stories with acceptance criteria, filed as issues.
- **`/retro`** — chronicler: read the last ~2 weeks of LOG + git history +
  `needs-replan` issues + verdict comments on issue threads; find
  recurring friction and classify it by pipeline entry point; audit the
  follow-up ledger (every promised follow-up filed or dismissed); apply
  the smallest durable fix to WORKFLOW/STYLE/agent prompts in a
  `meta: retro` commit; log metric trends vs the previous retro.
  Then Part 2, the **architecture audit** — delegate to the
  **steward**: import graph and exported surface vs
  docs/ARCHITECTURE.md; placement, duplicate concepts/representations
  (judged by upkeep cost), exported-symbol usage vs godoc intent, doc
  drift. Files `kind/refactor` issues ranked by cost-of-delay;
  pre-1.0 movement is encouraged, post-1.0 the audit guards the
  surface (docs/PLAN.md defines the line).
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
