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

2. **Pick** — partition `wip/*` by lease: a branch whose tip
   (`git log -1 --format=%cI origin/wip/issue-<N>`) is newer than the
   **2h claim TTL** is LIVE — off-limits, along with its issue; older is
   EXPIRED — resumable.
   - **Resuming beats starting**: oldest EXPIRED `wip/issue-<N>` whose
     issue is open and not `needs-replan` → `git switch wip/issue-<N>`,
     merge `origin/main` in if main moved — never rebase (non-trivial
     conflicts → park it, comment, pick again), read the issue's newest
     `RESUME:` comment, continue from its "Next:" at the matching step
     below.
   - Otherwise take the highest-priority `ready` issue with closed
     dependencies and no live branch:
     `git switch -c wip/issue-<N> origin/main` and
     `git push -u origin HEAD` — the push is the claim. Push rejected →
     lost the race; fetch and pick again.
   - Neither → delegate to **cartographer** to plan, then STOP.

   Any rejected push to `wip/*` at any later step means another session
   holds the branch: fetch, abandon the local attempt, and stop (log
   it). NEVER force-push a `wip/*` or `parked/*` ref.

3. **Ground** — delegate spec questions to **oracle**. Post the answer
   verbatim as a `GROUNDING:` comment on the issue; save a copy to
   `.agent/grounding-issue-<N>.md` as scratch. CHECKPOINT
   (`git commit -am "wip #<N>: grounding" && git push`).

4. **Implement** — if the issue's `## Surface` section is non-"none",
   have **warden** review the PLANNED surface first (the Surface sketch
   plus mason's intended type shapes — a one-comment design pre-flight,
   posted on the issue) BEFORE any code is written; shape errors are
   cheapest before they're built. Then delegate to **mason**,
   committing on the WIP branch. Public API added/changed → **warden**
   reviews the diff too; post its verdict on the issue. CHECKPOINT.

5. **Judge** — delegate to **arbiter** (it reviews the branch diff
   against main). CHECKPOINT after each verdict. On reject: ONE repair
   round by mason (edit flagged lines only), re-judge. On a second
   reject: PARK — checkpoint the branch one final time, post the
   findings on the issue, relabel `needs-replan` (that alone retires
   the branch; nothing is renamed or deleted), then go to step 6's log
   entry (committed on main) and stop. Never solicit a third round.

6. **Land** — delegate the log entry to **chronicler** (docs/LOG on the
   WIP branch, BEFORE landing), checkpoint. Then land through GitHub,
   never a local merge: open a PR from `wip/issue-<N>` to `main`
   (`Closes #<N>` in the body) and squash-merge it via the Merge API
   with the CLAUDE.md commit format as squash title and body. GitHub
   lands the ONE session commit, closes the issue, and auto-deletes the
   branch.

If the session must end early at any point: CHECKPOINT + a `RESUME:` /
"Next:" comment on the issue is a successful hand-off. Budget: one issue
per session.
