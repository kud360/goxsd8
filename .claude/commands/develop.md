---
description: One develop iteration — survey the WIP index, resume or start an issue branch, ground, implement, judge, land. Stop after one issue.
---

One iteration, then stop. You are the orchestrator: you delegate all
specialist work, you never skip the arbiter, and you never wait for a
human — abort hanging commands and log the failure.

docs/WORKFLOW.md is normative for the branch scheme, checkpointing, scope,
landing and parking; this file is the sequence. **Checkpoint (commit +
push) at every step boundary** — it is both the crash guard and the lease
heartbeat.

1. **Survey.** `git config core.hooksPath .githooks` (idempotent, and a
   fresh clone carries no local config), then `git fetch --prune origin`
   — without `--prune` the local view keeps refs for branches GitHub
   deleted at merge. A dirty local tree
   gets pushed to `parked/untriaged-<YYYYMMDD-HHMMSS>` and logged — never
   cleaned.

2. **Pick.** Run `go tool wipsurvey` — CLAUDE.md spells its input — rather
   than re-deriving the branch namespace by hand (#399). It reports each
   `wip/issue-<N>` as LIVE, CLAIMED, EXPIRED, RETIRED or UNKNOWN. LIVE and
   CLAIMED branches and their issues are off-limits; RETIRED ones are not
   work; UNKNOWN wants `git fetch origin` and a re-run. A CLAIMED branch
   has pushed no commits of its own, so git dates nothing about it — only
   its issue thread's own clock settles whether the holder is still
   working, and no age reading may retire it (#722). If no channel yields
   an issue list, run it on empty stdin: the lease-only report classifies
   everything but RETIRED, which is not a reason to survey by hand.
   - **Resuming beats starting.** Take the oldest EXPIRED `wip/issue-<N>`
     whose issue is open and not `needs-replan`; merge `origin/main` in if
     main moved (never rebase; if the conflicts are not tractable, park it
     and pick again), read the newest `RESUME:` comment, continue from its
     "Next:" at the matching step below.
   - Otherwise claim the highest-priority `ready` issue with closed
     dependencies and no live branch:
     `git switch -c wip/issue-<N> origin/main && git push -u origin HEAD`.
     The push is the claim; a rejected push means you lost the race — fetch
     and pick again.
   - Nothing to resume and nothing ready → delegate to **cartographer**,
     then stop.

3. **Ground.** Search the open queue for this issue's primary file path
   and identifier and record what any hit was (duplicate or adjacent).
   Ask the **oracle** for the clauses and rule IDs in scope, and post its
   answer verbatim as a `GROUNDING:` comment — that comment is the only
   durable copy.

4. **Implement.** If the issue's `## Surface` is non-"none", have
   **warden** pre-flight the planned shape before any code exists; shape
   errors are cheapest before they are built. Then delegate to **mason**,
   always with worktree isolation. Only once mason reports completion,
   fast-forward-merge its local branch into `wip/issue-<N>`. If the change
   added or altered public API, warden reviews the diff too. Post both
   verdicts on the issue.

   Mason may absorb adjacent work under docs/WORKFLOW.md's scope rule.
   Absorbed items belong in the commit body, not in a new issue.

5. **Judge.** `git fetch origin main` first and merge it forward if it
   moved — before the arbiter, per WORKFLOW's **After the verdict**, where
   it costs no round instead of one. Then delegate to **arbiter**. On
   reject: one repair round by mason, then re-judge. On a second reject:
   park per WORKFLOW, then go to step 6's log entry and stop. On accept,
   dispose of the verdict's remaining findings per WORKFLOW's **After the
   verdict** before step 6.

6. **Land.** Delegate the log entry to **chronicler** first, so it rides
   the session commit. Verify each of WORKFLOW's **Landing** preconditions
   yourself, by the check it names, then open the PR and squash-merge it
   via the Merge API using CLAUDE.md's commit format. A PR may close more
   than one issue when a landing carried more than one.

7. **Post-land.** Delegate to **cartographer**: unblock whatever this
   landing unblocked, and dispose of every follow-up this session raised
   while it is still fresh.

Ending early at a checkpoint with a good `RESUME:` comment is a successful
session. Budget: one landing.
