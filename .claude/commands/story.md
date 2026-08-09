---
description: Generate user stories — this session interviews the libuser and cliuser personas against the current published surface, and the cartographer files them as issues.
---

1. Collect the current published surface: README.md, `go doc ./...`, and
   `goxsd8 -help` if the CLI builds.
2. Delegate to **libuser** and **cliuser** in character, giving each ONLY
   that surface. Each returns 2–4 concrete stories with the code or
   command lines they wish would work, acceptance criteria, and the
   documentation gaps they hit.
3. Delegate to **cartographer** to reconcile those against existing issues
   and file the survivors as `kind/story`. Documentation gaps are filed as
   bugs — the docs are the tested product surface (PRINCIPLES 31).
4. Delegate a log entry to **chronicler**, commit any doc fixes as
   `meta: story <date>`, and land via a PR squash-merged in this same
   session.
