---
description: One develop iteration — pick a ready issue, ground it in the spec, implement, judge, log, commit, push. Stop after one issue.
---

Run ONE develop iteration per docs/WORKFLOW.md. You are the orchestrator:
delegate all specialist work to the subagents; never skip the arbiter;
never wait for a human — abort hanging commands and log the failure.

1. **Rescue** — the container may be a fresh clone; anything not pushed
   does not exist. If the tree is somehow dirty (persistent local
   checkout), rescue it to a PUSHED branch — `git switch -c
   rescue/untriaged-<YYYYMMDD-HHMMSS>`, `git add -A`, commit, `git push
   -u origin HEAD`, `git switch main` — and note it for the log. NEVER
   `git clean`, `git restore .`, `git checkout -- <file>`, or any
   stashing. When picking an issue (step 2), check its newest comments
   for a RESUME note: if one names a rescue branch, `git fetch origin`,
   `git switch` to it (rebase onto origin/main if main moved), and
   continue from its "Next:"; land accepted work on main via
   `git merge --squash`, then delete the remote rescue branch.

2. **Pick** — list open issues labeled `ready`; take the highest-priority
   one whose dependencies are closed. If none exist, delegate to
   **cartographer** to plan, then STOP (no implementation this session).

3. **Ground** — delegate the issue's spec questions to **oracle**. Post
   the answer verbatim as a `GROUNDING:` comment on the issue; also save
   to `.agent/grounding-issue-<N>.md` as scratch.

4. **Implement** — delegate to **mason** with the issue number and the
   grounding. If the change adds or alters public API, delegate the
   design to **warden** first and post its verdict on the issue.

5. **Judge** — delegate to **arbiter**. On reject: ONE repair round by
   mason (edit flagged lines only), then re-judge. On a second reject:
   rescue the work to a pushed branch (`rescue/issue-<N>-<ts>`, per
   docs/WORKFLOW.md "Checkpoints & resume"), post the findings and a
   RESUME comment naming the branch, relabel `needs-replan`, go to
   step 6, and stop after it. Never solicit a third round.

6. **Record & commit** — delegate the log entry to **chronicler**
   (docs/LOG, BEFORE the commit). Then ONE commit carrying code + log,
   in the format from CLAUDE.md (`<area>: <what> (#<N>)`, `Spec:`,
   `Ratchet:` trailers). Close the issue (or comment why not), push.
   The tree must be clean after the push.

Budget: one issue per session. A pushed rescue branch plus a good issue
comment is a successful session.
