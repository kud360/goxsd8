---
description: Arbiter-only conformance maintenance — run the suite, report per-lane movement, ratchet upward, investigate regressions.
---

Delegate to the **arbiter**; no other agent touches the ratchet.

1. `git submodule update --init testdata/xsdtests` if absent, then run the
   read-only conformance check.
2. Report movement per lane (datatypes / schema / instance / xpath / json
   / ber).
3. Cases doing BETTER than expected → bank them per the arbiter's ratchet
   procedure and commit as
   `conformance: ratchet <date> (<lane movement>)`. Every flipped case
   must be explainable; an unexplained upward flip gets an issue before it
   gets committed.
4. Cases doing WORSE → expectations are not touched. Bisect to the causing
   commit, file a `kind/bug` issue with the case IDs and the suspect
   commit, and leave the failing gate as the alarm.
5. Delegate a log entry to **chronicler**, commit it with any lane
   updates, and land via a PR opened and squash-merged in this same
   session.
