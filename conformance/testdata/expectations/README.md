# Conformance expectations — machine-written, upward-only

One lane per file (`datatypes.txt`, `schema.txt`, `instance.txt`,
`xpath.txt`, `json.txt`, `ber.txt`), one case per line:

```
<case-id> <pass|fail>
```

Sorted by case ID. `#` starts a comment. `pass` = this processor agrees
with the suite's declared outcome; `fail` = a recorded known gap.

**Rules (constitutional — see CLAUDE.md):**

- Files are written only by the ratchet
  (`GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1`),
  never by hand.
- Scores only move up. The ratchet refuses to record any regression or
  vanished case; a change that would need one must be fixed or reverted.
- Never edit a file downward to make CI green.
- Expectation movement commits together with the change that earned it,
  and every flipped case must be explainable by that change's diff.
- **Sanctioned applicability removals are the one way a line leaves a
  file** (issue #576, repo-owner ruling). When suite discovery stops
  producing a case because the W3C suite's own `@version` applicability
  metadata scopes it away from an XSD 1.1 processor, that case was never
  this processor's to score: it is a sanctioned removal, not a `Vanished`
  regression, and banking it DELETES its line (tolerating it without
  deleting would re-offer and re-refuse the same case forever). Two locks
  guard it, and both must hold:
  1. **The runner classifies, not the agent.** Only a case ID discovery
     itself withheld can be removed. An expected case that merely stopped
     appearing is `Vanished` and still aborts the merge.
  2. **The arbiter asserts the count, per lane**
     (`GOXSD_RATCHET_REMOVALS=schema=34,instance=65`, ratchet runs only).
     Any other number refuses the entire merge, so "sanctioned" is a
     figure the machinery checks, never a claim made in prose.
- Nothing else is relaxed: scores still only move up, a genuine
  `Regressed` or `Vanished` case still aborts the merge whatever the
  removal assertion says, and this file is still machine-written only.

When a recorded `fail` is an honestly-declined gap rather than something
being fixed, its durable dismissal record is the `docs/LOG` entry for the
session that declined it (and/or a GitHub tracking-issue comment) — not a
`#` comment beside the case in the lane file. The ratchet's
`WriteExpectations` regenerates each lane purely from its status map on
every run and has no comment-emission channel, so any hand-added comment
is silently discarded on the next ratchet run. Do not ask for, or attempt,
a lane-file comment as the record of a declined case.

A missing lane file is an empty lane (the lane's first ratchet run
creates it).
