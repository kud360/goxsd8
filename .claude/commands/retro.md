---
description: Weekly process retrospective + architecture audit — recraft the process from what the evidence shows, then run the steward's drift review.
---

Two delegations, in order — process first, architecture second.

**Part 1 — process retro.** Delegate to the **chronicler** for its full
retro: read the window's evidence, find the friction that RECURS, audit
the follow-up ledger, and recraft the process documents rather than
appending to them. Land the result as a `meta: retro <date>` commit,
opened and squash-merged as a PR in this same session.

**Part 2 — architecture audit.** Delegate to the **steward** for its full
audit. Code findings become `kind/refactor` issues ranked by
cost-of-delay; doc corrections land in a `meta: audit <date>` commit in
the same session; code moves are never made here — they go through the
develop loop.

Neither agent can be assumed to hold a GitHub channel and the steward's
tools are read-only, so **this session files what they return** — tooling
candidates, ledger leaks, refactor findings — and posts anything handed
back for a thread. Budget for it: an audit that "filed nothing" is an
orchestration miss, not a steward finding.

Constitutional guardrail: CLAUDE.md's "one rule" and the arbiter's
ratchet-integrity section are NOT editable here. They change only via a
human-filed issue.
