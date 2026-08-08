---
description: Arbiter-only conformance maintenance — run the suite, report per-lane movement, ratchet upward, investigate regressions.
---

Delegate to the **arbiter** (no other agent touches the ratchet):

1. `git submodule update --init testdata/xsdtests` if absent, then run
   `go test ./conformance -run TestConformance -count=1`.
2. Report movement per lane (datatypes / schema / instance / xpath /
   json / ber).
3. Cases doing BETTER than expected → run
   `GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1`
   and commit the lane updates:
   `conformance: ratchet <date> (<lane movement>)`. Every flipped case
   must be explainable — an unexplained upward flip gets an issue before
   it gets committed.
4. Cases doing WORSE → do NOT touch expectations. Bisect to the causing
   commit, file a `kind/bug` issue with the case IDs and the suspect
   commit, and leave the failing gate as the alarm. A case the read-only
   run logged as a **sanctioned applicability removal** (issue #576) is
   the exception: verify each ID's `@version` justification against the
   diff, then bank them by adding the per-lane count to the ratchet run,
   `GOXSD_RATCHET_REMOVALS=schema=<n>,instance=<n>`, and enumerate the
   removed IDs with their justification in the verdict. Any other number
   refuses the merge — never adjust the assertion to match a surprise.
5. Delegate a log entry to **chronicler**, commit (log rides with any
   lane updates), then land it via a PR opened and squash-merged in the
   same session (as `wip/issue-` work lands), not just a push.
