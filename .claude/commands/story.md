---
description: Generate user stories — cartographer interviews the libuser and cliuser personas against the current published surface and files them as issues.
---

Run a story session per docs/WORKFLOW.md:

1. Collect the current published surface: README.md and
   `go doc ./...` output (plus `goxsd8 -help` if the CLI builds).
2. Delegate to **libuser** and **cliuser** IN CHARACTER, giving each
   ONLY that surface: have each produce 2–4 concrete stories with the
   code snippets / command lines they wish would work, plus acceptance
   criteria and any documentation gaps they hit.
3. Delegate to **cartographer** to reconcile the stories against
   existing issues (dedupe, split, attach to milestones) and file the
   new ones as `kind/story` issues using the standard body template.
   Documentation gaps the personas hit are filed as bugs
   (PRINCIPLES 31 — the docs are the tested product surface).
4. Delegate a log entry to **chronicler**, commit any doc fixes that
   fell out (`meta: story <date>`), push.
