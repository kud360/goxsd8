---
description: Reconcile GitHub issues with reality and keep 8–10 ready issues, consulting the user personas for API/CLI-facing work. Doc-only commit; no code changes.
---

Run the **cartographer**'s full /backlog procedure (see its agent file and
docs/WORKFLOW.md): survey reality (git log, docs/LOG, issues, ratchet
lanes, `grep -rn "GAP("`), reconcile the issue list, keep 8–10 `ready`
issues with complete bodies, order by dependency, and update
docs/PLAN.md on drift.

For API- or CLI-facing milestones, have the cartographer consult
**libuser** and **cliuser** (feed them only the current README and
`go doc` output) and fold their stories and acceptance criteria into the
issue bodies.

Then delegate a session log entry to **chronicler**, commit any PLAN.md/
doc edits (`meta: backlog <date>`), and land them the same way `wip/issue-`
work lands — open a PR and squash-merge it via the GitHub Merge API in this
same session (never leave the commit sitting on an unmerged branch). No code
changes in this trigger.
