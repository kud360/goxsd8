---
description: Reconcile GitHub issues with reality and keep 5–10 ready issues, consulting the user personas for API/CLI-facing work. Doc-only commit; no code changes.
---

Run the **cartographer**'s full /plan procedure (see its agent file and
docs/WORKFLOW.md): survey reality (git log, docs/LOG, issues, ratchet
lanes, `grep -rn "GAP("`), reconcile the issue list, keep 5–10 `ready`
issues with complete bodies, order by dependency, and update docs/PLAN.md
on drift.

For API- or CLI-facing milestones, have the cartographer consult
**libuser** and **cliuser** (feed them only the current README and
`go doc` output) and fold their stories and acceptance criteria into the
issue bodies.

Then delegate a session log entry to **chronicler**, commit any PLAN.md/
doc edits (`meta: plan <date>`), and push. No code changes in this
trigger.
