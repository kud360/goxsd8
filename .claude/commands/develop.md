---
description: One develop iteration — survey the WIP index, resume or start an issue branch, ground, implement, judge, land. Stop after one issue.
---

Run ONE develop iteration per docs/WORKFLOW.md (the branch scheme section
is normative). You are the orchestrator: delegate all specialist work to
the subagents; never skip the arbiter; never wait for a human — abort
hanging commands and log the failure. The container is ephemeral:
**anything not pushed does not exist** — checkpoint (commit + push the
WIP branch) at every step boundary.

1. **Survey** — `git fetch origin` and list in-flight work:
   `git ls-remote --heads origin 'refs/heads/wip/*'`. If the local tree
   is somehow dirty, push it to `parked/untriaged-<YYYYMMDD-HHMMSS>` and
   log it. NEVER `git clean`, `git restore .`, `git checkout -- <file>`,
   or any stashing.

2. **Pick — resuming beats starting.**
   - A `wip/issue-<N>` exists with its issue open and not
     `needs-replan` → `git switch wip/issue-<N>`, rebase onto
     `origin/main` if main moved (non-trivial conflicts → park it,
     comment, pick again), read the issue's newest `RESUME:` comment,
     continue from its "Next:" at the matching step below.
   - Otherwise take the highest-priority `ready` issue with closed
     dependencies: `git switch -c wip/issue-<N> origin/main` and
     `git push -u origin HEAD` — the push is the claim.
   - Neither → delegate to **cartographer** to plan, then STOP.

3. **Ground** — delegate spec questions to **oracle**. Post the answer
   verbatim as a `GROUNDING:` comment on the issue; save a copy to
   `.agent/grounding-issue-<N>.md` as scratch. CHECKPOINT
   (`git commit -am "wip #<N>: grounding" && git push`).

4. **Implement** — delegate to **mason**, committing on the WIP branch.
   Public API added/changed → **warden** reviews first; post its verdict
   on the issue. CHECKPOINT.

5. **Judge** — delegate to **arbiter** (it reviews the branch diff
   against main). CHECKPOINT after each verdict. On reject: ONE repair
   round by mason (edit flagged lines only), re-judge. On a second
   reject: PARK —

   ```sh
   git push origin wip/issue-<N>:refs/heads/parked/issue-<N>-<ts>
   git push origin --delete wip/issue-<N>
   ```

   — post findings + the parked branch name on the issue, relabel
   `needs-replan`, then go to step 6's log entry (committed on main)
   and stop. Never solicit a third round.

6. **Land** — delegate the log entry to **chronicler** (docs/LOG on the
   WIP branch, BEFORE landing). Then atomically:

   ```sh
   git switch main && git pull --ff-only
   git merge --squash wip/issue-<N>
   git commit    # ONE commit: code + log, CLAUDE.md format
   git push origin main
   git push origin --delete wip/issue-<N>
   ```

   Close the issue. If anything fails between the main push and the
   branch delete, the leftover branch is safe — the cartographer's GC
   reconciles it.

If the session must end early at any point: CHECKPOINT + a `RESUME:` /
"Next:" comment on the issue is a successful hand-off. Budget: one issue
per session.
