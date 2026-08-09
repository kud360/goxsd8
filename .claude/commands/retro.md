---
description: Weekly process retrospective + architecture audit — recraft the process from what the evidence shows, then run the steward's drift review.
---

Two delegations, in order — process first, architecture second.

**Part 1 — process retro.** Delegate to the **chronicler**.

Read the evidence: the last ~2 weeks of docs/LOG, `git log`,
`needs-replan` issues, and the issue threads themselves — repair rounds
and advisory notes live in verdict comments, and the log under-reports
them. Find friction that RECURS; two sessions hitting the same wall is a
pattern, one session is an anecdote. Audit the follow-up ledger: every
"Next:", advisory note and promised follow-up either has a tracking issue
or gets explicitly dismissed.

Then **recraft, do not append.** Take the whole process as it now stands
— WORKFLOW, the command files, the agent files, STYLE, PRINCIPLES — and
ask what it should say given everything now known, not what the smallest
edit would be. A rule earned this week gets integrated into the text that
governs its subject, which usually means rewriting that section so it
reads as though the rule had always been there. Retire what the new
understanding supersedes.

The failure mode this exists to prevent is accretion: a rule appended per
incident, each carrying its own case history, until the documents are too
long to read and agents grep them instead. Prefer one clear sentence to a
paragraph of provenance — the evidence belongs in docs/LOG and on the
issue thread, and a rule needs at most a bare `(#N)` pointer to it. If a
section has grown to the point where nobody would read it, that is itself
the finding, and rewriting it is the fix.

Keep the ownership boundaries in CLAUDE.md's ground-truth table intact:
each document owns its subject and none restates another. A fix that has
to be stated in two places is a sign it belongs in neither.

Repeated manual toil becomes a `kind/tooling` issue instead of a rule
(PRINCIPLES 27). Land the result as a `meta: retro <date>` commit, opened
and squash-merged as a PR in this same session, and log the metric trends
against the previous retro: sessions per commit, repair rounds per accept,
rejects per accept, ratchet slope, ready-queue depth.

**Part 2 — architecture audit.** Delegate to the **steward** for its full
audit. Code findings become `kind/refactor` issues ranked by cost-of-delay;
doc corrections land in a `meta: audit <date>` commit in the same session;
code moves are never made here — they go through the develop loop.

The steward cannot file issues or merge its own PR (read-only tools, and
GitHub has been unreachable from that context on every audit so far), so
**the orchestrating session files what it returns and lands its commit.**
Budget for that step: an audit that "filed nothing" is an orchestration
miss, not a steward finding.

Constitutional guardrail: CLAUDE.md's "one rule" and the arbiter's
ratchet-integrity section are NOT editable here. They change only via a
human-filed issue.
