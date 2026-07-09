---
description: Weekly process retrospective + architecture audit — mine the log and git history for recurring friction, apply the smallest durable fix, then run the steward's drift review.
---

Two delegations, in order — process first, architecture second.

**Part 1 — process retro.** Delegate to the **chronicler**'s /retro duty
(see its agent file):

1. Read the last ~2 weeks of docs/LOG, `git log`, `needs-replan`
   issues, and the issue threads' verdict comments (repair rounds and
   advisory notes live there, not in the log).
2. Identify RECURRING friction (two sessions hitting the same wall is a
   pattern; one-offs are not) and classify each pattern by where it
   enters the pipeline — the fix belongs at the entry point.
3. Audit the follow-up ledger: every "Next:", advisory note, and
   promised follow-up in the window has a tracking issue or an explicit
   dismissal; file the leaks.
4. Apply the smallest durable fix — an edit to docs/WORKFLOW.md,
   docs/STYLE.md, an agent prompt, a command file, or a new
   docs/PRINCIPLES.md entry — in a dedicated `meta: retro <date>` commit.
   Repeated manual toil becomes a `kind/tooling` issue instead.
5. Log metric trends vs the previous retro (sessions/commit, repair
   rounds/accept, rejects/accept, ratchet slope, ready-queue depth)
   in the session log entry, commit, push.

**Part 2 — architecture audit.** Delegate to the **steward** (its
full audit procedure — see its agent file): import graph and exported
surface vs docs/ARCHITECTURE.md, placement, duplicate concepts and
representations (judged by upkeep cost, not existence), exported-symbol
usage vs godoc intent, doc drift. Code findings become `kind/refactor`
issues ranked by cost-of-delay — pre-1.0, movement is encouraged;
post-1.0 (human-declared, docs/PLAN.md) the audit guards the surface
instead. Doc corrections land in their own `meta: audit <date>` commit
in the same session; code moves are NEVER made here — they go through
the develop loop as issues.

Constitutional guardrail: the "one rule" in CLAUDE.md and the arbiter's
ratchet-integrity section are NOT editable here — changes to those need a
human-filed issue.
