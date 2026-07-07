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

A missing lane file is an empty lane (the lane's first ratchet run
creates it).
