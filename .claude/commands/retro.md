---
description: Weekly process retrospective — mine the log and git history for recurring friction and apply the smallest durable fix.
---

Delegate to the **chronicler**'s /retro duty (see its agent file):

1. Read the last ~2 weeks of docs/LOG, `git log`, and `needs-replan`
   issues.
2. Identify RECURRING friction (two sessions hitting the same wall is a
   pattern; one-offs are not).
3. Apply the smallest durable fix — an edit to docs/WORKFLOW.md,
   docs/STYLE.md, an agent prompt, a command file, or a new
   docs/PRINCIPLES.md entry — in a dedicated `meta: retro <date>` commit.
   Repeated manual toil becomes a `kind/tooling` issue instead.
4. Log one metric trend (sessions/commit, rejects/accept, ratchet slope)
   in the session log entry, commit, push.

Constitutional guardrail: the "one rule" in CLAUDE.md and the arbiter's
ratchet-integrity section are NOT editable here — changes to those need a
human-filed issue.
